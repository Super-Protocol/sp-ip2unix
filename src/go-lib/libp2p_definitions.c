#include "libp2p_definitions.h"

void NewStream(OnStreamCallback f,  voidPtr additionalParams, Libp2pStream* stream)
{
    f(additionalParams, stream);
}