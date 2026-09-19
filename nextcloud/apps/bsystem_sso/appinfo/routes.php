<?php

declare(strict_types=1);

return [
    'routes' => [
        [
            'name' => 'login#login',
            'url' => '/login',
            'verb' => 'GET',
        ],
        [
            'name' => 'login#entry',
            'url' => '/entry',
            'verb' => 'GET',
        ],
    ],
];
