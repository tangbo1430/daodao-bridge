<?php

namespace app\imcallback\projector;

class NoopProjector implements ProjectorInterface
{
    public function project(array $event, array $payload)
    {
    }
}
