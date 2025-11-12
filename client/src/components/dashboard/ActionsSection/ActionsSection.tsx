import { Card } from "@components/common/ui/Card/Card";
import { type Component, For } from "solid-js";
import styles from "./ActionsSection.module.scss";

export interface ActionItem {
  title: string;
  description: string;
  buttonText: string;
  onClick: () => void;
  disabled?: boolean;
  highlight?: boolean;
}

interface ActionsSectionProps {
  actions: ActionItem[];
}

export const ActionsSection: Component<ActionsSectionProps> = (props) => {
  const handleKeyDown = (event: KeyboardEvent, action: ActionItem) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      if (!action.disabled) {
        action.onClick();
      }
    }
  };

  return (
    <section class={styles.actionsSection}>
      <div class={styles.cardGrid}>
        <For each={props.actions}>
          {(action) => (
            <Card
              class={`${styles.actionCardContent} ${action.highlight ? styles.highlightCard : ""}`}
              onClick={action.disabled ? undefined : action.onClick}
            >
              {/* biome-ignore lint/a11y/useSemanticElements: Using div to avoid nested buttons (card contains button inside) */}
              <div
                role="button"
                tabIndex={action.disabled ? -1 : 0}
                onKeyDown={(e) => handleKeyDown(e, action)}
                class={styles.cardInteractive}
              >
                <div class={styles.cardHeader}>
                  <h2>{action.title}</h2>
                </div>
                <div class={styles.cardBody}>
                  <p>{action.description}</p>
                </div>
                <div class={styles.cardFooter}>
                  <button
                    type="button"
                    class={styles.button}
                    onClick={(e) => {
                      e.stopPropagation();
                      action.onClick();
                    }}
                    disabled={action.disabled}
                    tabIndex={-1}
                  >
                    {action.buttonText}
                  </button>
                </div>
              </div>
            </Card>
          )}
        </For>
      </div>
    </section>
  );
};
