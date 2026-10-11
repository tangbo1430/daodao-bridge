<?php

namespace app\imcallback\projector;

interface ProjectorInterface
{
    public function project(array $event, array $payload);
}
