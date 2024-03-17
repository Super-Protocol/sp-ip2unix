#pragma once

#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

typedef const char libp2p_const_char;

typedef struct Libp2pHandle {
    size_t _handle;
} Libp2pHandle;

typedef struct Libp2pError {
    int32_t code;
    char *message;
} Libp2pError;

typedef struct Libp2pHost {
    size_t _handle;
} Libp2pHost;

typedef struct Libp2pStream {
    size_t _handle;
} Libp2pStream;


typedef struct Libp2pHostResult {
    Libp2pHost *host;
    Libp2pError *error;
} Libp2pHostResult;

typedef struct Libp2pPublicAddressResult {
    char *address;
    Libp2pError *error;
} Libp2pPublicAddressResult;

typedef struct Libp2pOpenStreamResult {
    Libp2pStream *stream;
    Libp2pError *error;
} Libp2pOpenStreamResult;


typedef struct Libp2pReadResult {
    size_t bytes_read;
    Libp2pError *error;
} Libp2pReadResult;

typedef struct Libp2pStringResult {
    const char* string;
    Libp2pError *error;
} Libp2pStringResult;


typedef void* voidPtr;

typedef void (*OnStreamCallback) (voidPtr additionalParams, Libp2pStream* stream);
void NewStream(OnStreamCallback f, voidPtr additionalParams, Libp2pStream* stream);

typedef void (*OnRecvMsg) (const char *peerId, uint8_t *data, uint32_t dataSize);
void NewMsg(OnRecvMsg f, const char *peerId, uint8_t  *data, uint32_t dataSize);