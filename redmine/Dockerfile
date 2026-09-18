FROM redmine:6.1.4

ARG REDMINE_OAUTH_REF=v4.2.2

USER root

RUN export DEBIAN_FRONTEND=noninteractive \
 && apt-get update \
 && apt-get install -y --no-install-recommends git build-essential \
 && rm -rf /var/lib/apt/lists/*

RUN chown -R redmine:redmine /usr/local/bundle \
 && chmod -R o-w /usr/local/bundle

RUN set -eu; \
    mkdir -p /usr/src/redmine/plugins/redmine_oauth; \
    cd /usr/src/redmine/plugins/redmine_oauth; \
    git init; \
    git remote add origin https://github.com/kontron/redmine_oauth.git; \
    git fetch --depth 1 origin "$REDMINE_OAUTH_REF"; \
    git checkout --detach FETCH_HEAD; \
    test "$(git rev-parse HEAD)" = "69f9a2a7b4f64a6aa6b5ffb5e4161891395f3310"; \
    rm -rf .git

WORKDIR /usr/src/redmine

# The image entrypoint runs as the redmine user. Running Bundler as root here
# leaves /home/redmine/.bundle root-owned and causes the runtime entrypoint to
# fail with exit code 23 when it refreshes the compact index.
RUN rm -rf /home/redmine/.bundle \
 && gosu redmine bundle install

# The upstream entrypoint explicitly supports arbitrary/non-root users. The
# build above leaves the Redmine tree and Bundler state writable by this user,
# so the runtime does not need to begin as root merely to drop privileges.
USER redmine
