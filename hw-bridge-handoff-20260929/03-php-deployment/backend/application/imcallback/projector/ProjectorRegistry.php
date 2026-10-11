<?php

namespace app\imcallback\projector;

class ProjectorRegistry
{
    public static function resolve($category)
    {
        switch ($category) {
            case 'group': return new GroupProjector();
            case 'relation': return new RelationProjector();
            case 'profile': return new ProfileProjector();
            case 'state': return new StateProjector();
            case 'message': return new NoopProjector();
            default: return null;
        }
    }
}
