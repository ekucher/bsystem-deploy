<?php

declare(strict_types=1);

namespace OCA\BsystemSso\Controller;

use OCP\AppFramework\Controller;
use OCP\AppFramework\Http\Attribute\NoCSRFRequired;
use OCP\AppFramework\Http\Attribute\PublicPage;
use OCP\AppFramework\Http\RedirectResponse;
use OCP\IRequest;
use OCP\IURLGenerator;
use OCP\IUserSession;

final class LoginController extends Controller {
    private const PROVIDER_ID = 1;
    private const DEFAULT_TARGET = '/apps/dashboard/';

    public function __construct(
        IRequest $request,
        private IUserSession $userSession,
        private IURLGenerator $urlGenerator,
    ) {
        parent::__construct('bsystem_sso', $request);
    }

    /**
     * BSYSTEM-HUB launch endpoint.
     *
     * Synchronize the local Nextcloud session with the current
     * central OIDC identity and return to the Nextcloud dashboard.
     */
    #[PublicPage]
    #[NoCSRFRequired]
    public function login(): RedirectResponse {
        return $this->startOidcSynchronization(self::DEFAULT_TARGET);
    }

    /**
     * Direct browser entry endpoint.
     *
     * This endpoint is intended for a new browser entry into Nextcloud.
     * It must not be applied globally to WebDAV, OCS or API requests.
     */
    #[PublicPage]
    #[NoCSRFRequired]
    public function entry(): RedirectResponse {
        return $this->startOidcSynchronization(self::DEFAULT_TARGET);
    }

    private function startOidcSynchronization(string $target): RedirectResponse {
        if ($this->userSession->isLoggedIn()) {
            $this->userSession->logout();
        }

        $oidcLogin = $this->urlGenerator->linkToRoute(
            'user_oidc.login.login',
            [
                'providerId' => self::PROVIDER_ID,
                'redirectUrl' => $target,
            ],
        );

        return new RedirectResponse($oidcLogin);
    }
}
