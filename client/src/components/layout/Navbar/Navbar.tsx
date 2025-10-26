import { FolderSelector } from "@components/folders/FolderSelector";
import { ROUTES } from "@constants/api.constants";
import { useAuth } from "@context/AuthContext";
import { A } from "@solidjs/router";
import { type Component, Match, Switch } from "solid-js";
import styles from "./NavBar.module.scss";

export const NavBar: Component = () => {
  const { isAuthenticated, logout } = useAuth();

  return (
    <nav class={styles.navbar}>
      <div class={styles.navbarContainer}>
        <div class={styles.navbarLogo}>
          <A href={ROUTES.HOME} class={styles.navbarTitle}>
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
            <Switch>
              <Match when={isAuthenticated()}>
                <li class={styles.navbarItem}>
                  <A href={ROUTES.PROFILE} class={styles.navbarLink} activeClass={styles.active}>
                    Profile
                  </A>
                </li>
                <li class={styles.navbarItem}>
                  <A href={ROUTES.COLLECTION} class={styles.navbarLink} activeClass={styles.active}>
                    Collection
                  </A>
                </li>
                <li class={styles.navbarItem}>
                  <A href={ROUTES.LOG_PLAY} class={styles.navbarLink} activeClass={styles.active}>
                    Log Play
                  </A>
                </li>
                <li class={styles.navbarItem}>
                  <A
                    href={ROUTES.PLAY_HISTORY}
                    class={styles.navbarLink}
                    activeClass={styles.active}
                  >
                    Play History
                  </A>
                </li>
                <li class={styles.navbarItem}>
                  <A href={ROUTES.EQUIPMENT} class={styles.navbarLink} activeClass={styles.active}>
                    Equipment
                  </A>
                </li>
              </Match>
            </Switch>
          </ul>

          {/* Folder Selector for authenticated users */}
          <Switch>
            <Match when={isAuthenticated()}>
              <div class={styles.navbarFolderSelector}>
                <FolderSelector navbar />
              </div>
            </Match>
          </Switch>

          <ul class={styles.navbarActions}>
            <li class={styles.navbarItem}>
              <Switch>
                <Match when={!isAuthenticated()}>
                  <A href={ROUTES.LOGIN} class={styles.navbarLink} activeClass={styles.active}>
                    Login
                  </A>
                </Match>
                <Match when={isAuthenticated()}>
                  <A href={ROUTES.HOME} class={styles.navbarLink} onClick={logout}>
                    Logout
                  </A>
                </Match>
              </Switch>
            </li>
          </ul>
        </div>
      </div>
    </nav>
  );
};
