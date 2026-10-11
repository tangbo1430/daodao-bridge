FROM php:8.0-fpm-bullseye

RUN apt-get update && apt-get install -y --no-install-recommends \
        libfreetype6-dev libjpeg62-turbo-dev libpng-dev libzip-dev libonig-dev unzip \
    && docker-php-ext-configure gd --with-freetype --with-jpeg \
    && docker-php-ext-install -j"$(nproc)" pdo_mysql mysqli mbstring gd zip bcmath \
    && pecl install redis-6.0.2 \
    && docker-php-ext-enable redis \
    && rm -rf /var/lib/apt/lists/*

COPY docker/php/php.ini /usr/local/etc/php/conf.d/99-app.ini
COPY docker/php/app-entrypoint.sh /usr/local/bin/app-entrypoint
COPY docker/php/app-preflight.php /opt/app-preflight.php
RUN chmod +x /usr/local/bin/app-entrypoint

WORKDIR /var/www/app
ENTRYPOINT ["app-entrypoint"]
CMD ["php-fpm"]
