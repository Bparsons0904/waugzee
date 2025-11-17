import MenuIcon from "@components/icons/MenuIcon";
import VinylIcon from "@components/icons/VinylIcon";
import { XIcon } from "@components/icons/XIcon";
import { ROUTES } from "@constants/api.constants";
import { useAuth } from "@context/AuthContext";
import { useSyncStatus } from "@context/SyncStatusContext";
import { useUserData } from "@context/UserDataContext";
import { A } from "@solidjs/router";
import { type Component, createSignal, Match, Show, Switch } from "solid-js";
import styles from "./NavBar.module.scss";
import { SyncStatusIndicator } from "./SyncStatusIndicator";

export const NavBar: Component = () => {
  const { isAuthenticated, logout } = useAuth();
  const { user } = useUserData();
  const { isSyncing } = useSyncStatus();
  const [isMobileMenuOpen, setIsMobileMenuOpen] = createSignal(false);

  const needsDiscogsKey = () => !user()?.configuration?.discogsToken;

  const toggleMobileMenu = () => {
    setIsMobileMenuOpen(!isMobileMenuOpen());
  };

  const closeMobileMenu = () => {
    setIsMobileMenuOpen(false);
  };

  return (
    <nav class={styles.navbar}>
      <div class={styles.navbarContainer}>
        <div class={styles.navbarHeader}>
          <div class={styles.navbarLogo}>
            <A href={ROUTES.HOME} class={styles.navbarTitle} onClick={closeMobileMenu}>
              <img src="/images/waugzee_logo.png" alt="Waugzee" class={styles.vinylLogo} />
              {/* <VinylIcon size={42} class={styles.vinylIcon} /> */}
              Waugzee
            </A>
          </div>
          <button
            type="button"
            class={styles.mobileMenuToggle}
            onClick={toggleMobileMenu}
            aria-label={isMobileMenuOpen() ? "Close menu" : "Open menu"}
            aria-expanded={isMobileMenuOpen()}
          >
            {isMobileMenuOpen() ? <XIcon size={24} /> : <MenuIcon size={24} />}
          </button>
        </div>

        <div class={styles.navbarMenu} classList={{ [styles.mobileMenuOpen]: isMobileMenuOpen() }}>
          <ul class={styles.navbarItems}>
            <li class={styles.navbarItem}>
              <A
                href="/"
                class={styles.navbarLink}
                activeClass={styles.active}
                end
                onClick={closeMobileMenu}
              >
                Home
              </A>
            </li>
            <Show when={isAuthenticated()}>
              <li class={styles.navbarItem}>
                <A
                  href={ROUTES.LOG_PLAY}
                  class={styles.navbarLink}
                  activeClass={styles.active}
                  onClick={closeMobileMenu}
                >
                  Log Play
                </A>
              </li>
              <li class={styles.navbarItem}>
                <A
                  href={ROUTES.COLLECTION}
                  class={styles.navbarLink}
                  activeClass={styles.active}
                  onClick={closeMobileMenu}
                >
                  Collection
                </A>
              </li>
              <li class={styles.navbarItem}>
                <A
                  href={ROUTES.PLAY_HISTORY}
                  class={styles.navbarLink}
                  activeClass={styles.active}
                  onClick={closeMobileMenu}
                >
                  Play History
                </A>
              </li>
              <Show when={user()?.isAdmin}>
                <li class={styles.navbarItem}>
                  <A
                    href={ROUTES.ADMIN}
                    class={styles.navbarLink}
                    activeClass={styles.active}
                    onClick={closeMobileMenu}
                  >
                    Admin
                  </A>
                </li>
              </Show>
            </Show>
          </ul>

          <ul class={styles.navbarActions}>
            <Show when={isAuthenticated()}>
              <li class={styles.navbarItem}>
                <SyncStatusIndicator isSyncing={isSyncing()} />
              </li>
            </Show>
            <Switch>
              <Match when={!isAuthenticated()}>
                <li class={styles.navbarItem}>
                  <A
                    href={ROUTES.LOGIN}
                    class={`${styles.navbarLink} ${styles.authButton}`}
                    activeClass={styles.active}
                    onClick={closeMobileMenu}
                  >
                    Login
                  </A>
                </li>
              </Match>
              <Match when={isAuthenticated()}>
                <li class={styles.navbarItem}>
                  <A
                    href={ROUTES.PROFILE}
                    class={`${styles.navbarLink} ${needsDiscogsKey() ? styles.needsAttention : ""}`}
                    activeClass={styles.active}
                    onClick={closeMobileMenu}
                  >
                    Profile
                  </A>
                </li>
                <li class={styles.navbarItem}>
                  <A
                    href={ROUTES.HOME}
                    class={`${styles.navbarLink} ${styles.authButton}`}
                    onClick={() => {
                      closeMobileMenu();
                      logout();
                    }}
                  >
                    Logout
                  </A>
                </li>
              </Match>
            </Switch>
          </ul>
        </div>
      </div>
    </nav>
  );
};
