import VinylIcon from "@components/icons/VinylIcon";
import { ROUTES } from "@constants/api.constants";
import { useAuth } from "@context/AuthContext";
import { useSyncStatus } from "@context/SyncStatusContext";
import { useUserData } from "@context/UserDataContext";
import { A } from "@solidjs/router";
import { type Component, Match, Show, Switch } from "solid-js";
import styles from "./NavBar.module.scss";
import { SyncStatusIndicator } from "./SyncStatusIndicator";

export const NavBar: Component = () => {
  const { isAuthenticated, logout } = useAuth();
  const { user } = useUserData();
  const { isSyncing } = useSyncStatus();

  const needsDiscogsKey = () => !user()?.configuration?.discogsToken;

  return (
    <nav class={styles.navbar}>
      <div class={styles.navbarContainer}>
        <div class={styles.navbarLogo}>
          <A href={ROUTES.HOME} class={styles.navbarTitle}>
            <VinylIcon size={42} class={styles.vinylIcon} />
            Waugzee
          </A>
        </div>
        <div class={styles.navbarMenu}>
          <ul class={styles.navbarItems}>
            <li class={styles.navbarItem}>
              <A href="/" class={styles.navbarLink} activeClass={styles.active} end>
                Home
              </A>
            </li>
            <Show when={isAuthenticated()}>
              <li class={styles.navbarItem}>
                <A href={ROUTES.LOG_PLAY} class={styles.navbarLink} activeClass={styles.active}>
                  Log Play
                </A>
              </li>
              <li class={styles.navbarItem}>
                <A href={ROUTES.COLLECTION} class={styles.navbarLink} activeClass={styles.active}>
                  Collection
                </A>
              </li>
              <li class={styles.navbarItem}>
                <A href={ROUTES.PLAY_HISTORY} class={styles.navbarLink} activeClass={styles.active}>
                  Play History
                </A>
              </li>
              <Show when={user()?.isAdmin}>
                <li class={styles.navbarItem}>
                  <A href={ROUTES.ADMIN} class={styles.navbarLink} activeClass={styles.active}>
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
                  >
                    Profile
                  </A>
                </li>
                <li class={styles.navbarItem}>
                  <A
                    href={ROUTES.HOME}
                    class={`${styles.navbarLink} ${styles.authButton}`}
                    onClick={logout}
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
