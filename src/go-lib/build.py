import subprocess
import os
import sys

script_dir = os.path.dirname(os.path.abspath(sys.argv[0]))
output = os.path.basename(os.path.join(sys.argv[1], sys.argv[2]))
# Run the script with the specified working directory
result = subprocess.run(['go', 'build', '-o' , output, '-buildmode=c-archive'], cwd=script_dir , stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
print("go-lib build {0} ret: {1}".format(output, result.returncode))
print(result.stdout)
print(result.stderr)
