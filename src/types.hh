// SPDX-License-Identifier: LGPL-3.0-only
#ifndef IP2UNIX_TYPES_HH
#define IP2UNIX_TYPES_HH

#include <unordered_map>
#include <string>

enum class SocketType {
    STREAM, DATAGRAM, INVALID
};

class OverlayRoute {
public:
    std::string ip;
    std::string id;
    int port;
    std::string realAddress;
};

using OverlayRouteMap = std::unordered_map<std::string, OverlayRoute>;

#endif
