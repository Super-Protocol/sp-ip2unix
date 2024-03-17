#include "libp2p_definitions.h"

void NewStream(OnStreamCallback f,  voidPtr additionalParams, Libp2pStream* stream)
{
    f(additionalParams, stream);
}

void NewMsg(OnRecvMsg f, const char *peerId, uint8_t *data, uint32_t dataSize)
{
    f(peerId, data, dataSize);
}