import { Component, For } from "solid-js";
import { Card } from "@components/common/ui/Card/Card";
import styles from "./ActionsSection.module.scss";

export interface ActionItem {
  title: string;
  description: string;
  buttonText: string;
  onClick: () => void;
}

interface ActionsSectionProps {
  actions: ActionItem[];
}

export const ActionsSection: Component<ActionsSectionProps> = (props) => {
  return (
    <section class={styles.actionsSection}>
      <div class={styles.cardGrid}>
        <For each={props.actions}>
          {(action) => (
            <Card class={styles.actionCardContent}>
              <div class={styles.cardHeader}>
                <h2>{action.title}</h2>
              </div>
              <div class={styles.cardBody}>
                <p>{action.description}</p>
              </div>
              <div class={styles.cardFooter}>
                <button class={styles.button} onClick={action.onClick}>
                  {action.buttonText}
                </button>
              </div>
            </Card>
          )}
        </For>
      </div>
    </section>
  );
};