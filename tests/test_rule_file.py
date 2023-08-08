import json
import subprocess
import sys
import unittest

from tempfile import NamedTemporaryFile

from helper import IP2UNIX, systemd_only, non_systemd_only, \
                   abstract_sockets_only


class RuleFileTest(unittest.TestCase):
    def assert_good_rules(self, rules):
        cmd = [IP2UNIX, '-c', '-F', json.dumps(rules)]
        result = subprocess.run(cmd, stdout=subprocess.PIPE,
                                stderr=subprocess.STDOUT)
        self.assertTrue(result.stdout.startswith(
            b'The use of -F/--rules-data option is deprecated and it will be'
            b' removed in ip2unix version 3. Please use the -r/--rule option'
            b' instead.\n'
        ))
        msg = 'Rules {!r} do not validate: {}'.format(rules, result.stdout)
        self.assertEqual(result.returncode, 0, msg)

    def assert_bad_rules(self, rules):
        cmd = [IP2UNIX, '-c', '-F', json.dumps(rules)]
        result = subprocess.run(cmd, stdout=subprocess.PIPE,
                                stderr=subprocess.STDOUT)
        self.assertTrue(result.stdout.startswith(
            b'The use of -F/--rules-data option is deprecated and it will be'
            b' removed in ip2unix version 3. Please use the -r/--rule option'
            b' instead.\n'
        ))
        msg = 'Rules {!r} should not be valid.'.format(rules)
        self.assertNotEqual(result.returncode, 0, msg)

    def test_no_array(self):
        self.assert_bad_rules({'rule1': {'socketPath': '/foo'}})
        self.assert_bad_rules({'rule2': {}})
        self.assert_bad_rules({})

    def test_empty(self):
        self.assert_good_rules([])

    def test_complete_rules(self):
        self.assert_good_rules([
            {'direction': 'outgoing',
             'type': 'udp',
             'socketPath': '/tmp/foo'},
            {'direction': 'incoming',
             'address': '::',
             'socketPath': '/tmp/bar'}
        ])

    def test_unknown_rule_attrs(self):
        self.assert_bad_rules([{'foo': 1}])
        self.assert_bad_rules([{'socketpath': 'xxx'}])

    def test_wrong_rule_types(self):
        self.assert_bad_rules([{'type': 'nope', 'socketPath': '/tmp/foo'}])
        self.assert_bad_rules([{'direction': 'out', 'socketPath': '/tmp/foo'}])
        self.assert_bad_rules([{'socketPath': 1234}])

    def test_no_socket_path(self):
        self.assert_bad_rules([{'address': '1.2.3.4'}])

    def test_relative_socket_path(self):
        self.assert_bad_rules([{'socketPath': 'aaa/bbb'}])
        self.assert_bad_rules([{'socketPath': 'bbb'}])

    def test_absolute_socket_path(self):
        self.assert_good_rules([{'socketPath': '/xxx'}])

    @abstract_sockets_only
    def test_valid_abstract_name(self):
        self.assert_good_rules([{'abstract': 'foobar'}])

    @abstract_sockets_only
    def test_invalid_abstract_name(self):
        self.assert_bad_rules([{'abstract': ''}])

    @abstract_sockets_only
    def test_abstract_and_path(self):
        self.assert_bad_rules([{'abstract': 'xxx', 'socketPath': '/xxx'}])

    def test_invalid_enums(self):
        self.assert_bad_rules([{'socketPath': '/bbb', 'direction': 111}])
        self.assert_bad_rules([{'socketPath': '/bbb', 'direction': False}])
        self.assert_bad_rules([{'socketPath': '/bbb', 'type': 234}])
        self.assert_bad_rules([{'socketPath': '/bbb', 'type': True}])

    def test_invalid_port_type(self):
        self.assert_bad_rules([{'socketPath': '/aaa', 'port': 'foo'}])
        self.assert_bad_rules([{'socketPath': '/aaa', 'port': True}])
        self.assert_bad_rules([{'socketPath': '/aaa', 'port': -1}])
        self.assert_bad_rules([{'socketPath': '/aaa', 'port': 65536}])

    def test_port_range(self):
        self.assert_good_rules([{'socketPath': '/aaa', 'port': 123,
                                 'portEnd': 124}])
        self.assert_good_rules([{'socketPath': '/aaa', 'port': 1000,
                                 'portEnd': 65535}])

    def test_invalid_port_range(self):
        self.assert_bad_rules([{'socketPath': '/aaa', 'port': 123,
                                'portEnd': 10}])
        self.assert_bad_rules([{'socketPath': '/aaa', 'port': 123,
                                'portEnd': 123}])
        self.assert_bad_rules([{'socketPath': '/aaa', 'port': 123,
                                'portEnd': 65536}])

    def test_missing_start_port_in_range(self):
        self.assert_bad_rules([{'socketPath': '/aaa', 'portEnd': 123}])

    def test_valid_address(self):
        valid_addrs = [
            '127.0.0.1', '0.0.0.0', '9.8.7.6', '255.255.255.255', '::',
            '::ffff:127.0.0.1', '7:6:5:4:3:2:1::', '::7:6:5:4:3:2:1',
            '7::', '::2:1', '::17', 'ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff'
        ]
        for addr in valid_addrs:
            self.assert_good_rules([{'socketPath': '/foo', 'address': addr}])

    def test_invalid_addrss(self):
        invalid_addrs = [
            '.0.0.1', '123', '123.', '..', '-1.2.3.4', '256.255.255.255',
            ':::', '0.00.0.0', '1.-2.3.4', 'abcde', '::-1', '01000::',
            'abcd::efgh', '8:7:6:5:4:3:2:1::', '::8:7:6:5:4:3:2:1',
            'f:f11::01100:2'
        ]
        for addr in invalid_addrs:
            self.assert_bad_rules([{'socketPath': '/foo', 'address': addr}])

    def test_valid_reject(self):
        for val in ["EBADF", "EINTR", "enomem", "EnOMeM", 13, 12]:
            self.assert_good_rules([{'reject': True, 'rejectError': val}])

    def test_invalid_reject(self):
        for val in ["EBAAAADF", "", "XXX", "vvv", -10]:
            self.assert_bad_rules([{'reject': True, 'rejectError': val}])

    def test_reject_with_sockpath(self):
        self.assert_bad_rules([{'socketPath': '/foo', 'reject': True}])

    def test_blackhole_with_reject(self):
        self.assert_bad_rules([{'direction': 'incoming', 'reject': True,
                                'blackhole': True}])

    def test_blackhole_outgoing(self):
        self.assert_bad_rules([{'blackhole': True}])
        self.assert_bad_rules([{'direction': 'outgoing', 'blackhole': True}])

    def test_blackhole_with_sockpath(self):
        self.assert_bad_rules([{'direction': 'incoming', 'socketPath': '/foo',
                                'blackhole': True}])

    def test_blackhole_all(self):
        self.assert_good_rules([{'direction': 'incoming', 'blackhole': True}])

    def test_ignore_with_sockpath(self):
        self.assert_bad_rules([{'socketPath': '/foo', 'ignore': True}])

    def test_ignore_with_reject(self):
        self.assert_bad_rules([{'reject': True, 'ignore': True}])

    def test_ignore_with_blackhole(self):
        self.assert_bad_rules([{'blackhole': True, 'ignore': True}])

    @systemd_only
    def test_ignore_with_systemd(self):
        self.assert_bad_rules([{'socketActivation': True, 'ignore': True}])

    @systemd_only
    def test_contradicting_systemd(self):
        self.assert_bad_rules([{'socketPath': '/foo',
                                'socketActivation': True}])

    @systemd_only
    def test_socket_fdname(self):
        self.assert_good_rules([{'socketActivation': True, 'fdName': 'foo'}])

    @non_systemd_only
    def test_no_systemd_options(self):
        self.assert_bad_rules([{'socketActivation': True}])
        self.assert_bad_rules([{'socketActivation': True, 'fdName': 'foo'}])

    def test_print_rules_check_stdout(self):
        rules = [
            {'direction': 'outgoing',
             'type': 'tcp',
             'socketPath': '/foo'},
            {'address': '0.0.0.0',
             'socketPath': '/bar'}
        ]
        cmd = [IP2UNIX, '-cp', '-F', json.dumps(rules)]
        result = subprocess.run(cmd, stderr=subprocess.PIPE,
                                stdout=subprocess.PIPE)
        self.assertEqual(result.returncode, 0)
        self.assertEqual(
            result.stderr,
            b'The use of -F/--rules-data option is deprecated and it will be'
            b' removed in ip2unix version 3. Please use the -r/--rule option'
            b' instead.\n'
        )
        self.assertNotEqual(result.stdout, b'')
        self.assertGreater(len(result.stdout), 0)
        self.assertIn(b'IP Type', result.stdout)

    def test_print_rules_stderr(self):
        rules = [{'socketPath': '/xxx'}]
        cmd = [IP2UNIX, '-p', '-F', json.dumps(rules),
               sys.executable, '-c', '']
        result = subprocess.run(cmd, stderr=subprocess.PIPE,
                                stdout=subprocess.PIPE)
        self.assertEqual(result.returncode, 0)
        self.assertEqual(result.stdout, b'')
        self.assertTrue(result.stderr.startswith(
            b'The use of -F/--rules-data option is deprecated and it will be'
            b' removed in ip2unix version 3. Please use the -r/--rule option'
            b' instead.\n'
        ))
        self.assertRegex(result.stderr[134:], b'^Rule #1.*')
        self.assertGreater(len(result.stderr), 0)
        self.assertIn(b'IP Type', result.stderr)

    def test_new_style_rule_files(self):
        with NamedTemporaryFile('w') as rf1, NamedTemporaryFile('w') as rf2:
            rf1.write('in,port=1234,path=/foo\n')
            rf1.flush()
            rf2.write('  \t in,addr=9.8.7.6,path=/bar\n'
                      # Note the second \n here is to make sure that we skip
                      # empty lines.
                      '# some comment\n\n'
                      # Only whitespace should be skipped as well.
                      '   \t  '
                      # Note: Missing \n is intentional here!
                      'out,port=4321,path=/foobar')
            rf2.flush()
            cmd = [IP2UNIX, '-c', '-p', '-f', rf1.name, '-f', rf2.name]
            result = subprocess.run(cmd, stdout=subprocess.PIPE,
                                    stderr=subprocess.STDOUT)

        self.assertEqual(
            result.stdout,
            b'Rule #1:\n'
            b'  Direction: incoming\n'
            b'  IP Type: TCP and UDP\n'
            b'  Address: <any>\n'
            b'  Port: 1234\n'
            b'  Socket path: /foo\n'
            b'Rule #2:\n'
            b'  Direction: incoming\n'
            b'  IP Type: TCP and UDP\n'
            b'  Address: 9.8.7.6\n'
            b'  Port: <any>\n'
            b'  Socket path: /bar\n'
            b'Rule #3:\n'
            b'  Direction: outgoing\n'
            b'  IP Type: TCP and UDP\n'
            b'  Address: <any>\n'
            b'  Port: 4321\n'
            b'  Socket path: /foobar\n'
        )
