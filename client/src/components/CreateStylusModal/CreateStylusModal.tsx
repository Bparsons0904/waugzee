import { Select } from "@components/common/forms/Select/Select";
import { TextInput } from "@components/common/forms/TextInput/TextInput";
import { Button } from "@components/common/ui/Button/Button";
import type { CreateCustomStylusRequest } from "@models/Stylus";
import { useCreateCustomStylus } from "@services/apiHooks";
import clsx from "clsx";
import type { Component } from "solid-js";
import { createStore } from "solid-js/store";
import styles from "./CreateStylusModal.module.scss";

interface CreateStylusModalProps {
  onClose: () => void;
}

interface FormState {
  brand: string;
  model: string;
  type: string;
  cartridgeType: string;
  recommendedReplaceHours?: number;
}

const CreateStylusModal: Component<CreateStylusModalProps> = (props) => {
  const createCustomStylusMutation = useCreateCustomStylus();

  const [formState, setFormState] = createStore<FormState>({
    brand: "",
    model: "",
    type: "",
    cartridgeType: "",
    recommendedReplaceHours: undefined,
  });

  const resetForm = () => {
    setFormState({
      brand: "",
      model: "",
      type: "",
      cartridgeType: "",
      recommendedReplaceHours: undefined,
    });
  };

  const handleClose = () => {
    resetForm();
    props.onClose();
  };

  const handleSubmit = (e: Event) => {
    e.preventDefault();
    if (!formState.brand || !formState.model) return;

    const request: CreateCustomStylusRequest = {
      brand: formState.brand,
      model: formState.model,
      type: formState.type || undefined,
      cartridgeType: formState.cartridgeType || undefined,
      recommendedReplaceHours: formState.recommendedReplaceHours,
    };

    createCustomStylusMutation.mutate(request, {
      onSuccess: () => {
        handleClose();
      },
    });
  };

  return (
    <form class={styles.form} onSubmit={handleSubmit}>
      <div class={styles.formRow}>
        <TextInput
          name="brand"
          label="Brand"
          value={formState.brand}
          onInput={(value) => setFormState("brand", value)}
          required
        />

        <TextInput
          name="model"
          label="Model"
          value={formState.model}
          onInput={(value) => setFormState("model", value)}
          required
        />
      </div>

      <div class={styles.formRow}>
        <Select
          name="type"
          label="Stylus Type"
          placeholder="-- Select type --"
          options={[
            { value: "Conical", label: "Conical" },
            { value: "Elliptical", label: "Elliptical" },
            { value: "Microline", label: "Microline" },
            { value: "Shibata", label: "Shibata" },
            { value: "Line Contact", label: "Line Contact" },
            { value: "Other", label: "Other" },
          ]}
          value={formState.type}
          onChange={(value) => setFormState("type", value)}
        />

        <Select
          name="cartridgeType"
          label="Cartridge Type"
          placeholder="-- Select cartridge type --"
          options={[
            { value: "Moving Magnet", label: "Moving Magnet (MM)" },
            { value: "Moving Coil", label: "Moving Coil (MC)" },
            { value: "Ceramic", label: "Ceramic" },
            { value: "Other", label: "Other" },
          ]}
          value={formState.cartridgeType}
          onChange={(value) => setFormState("cartridgeType", value)}
        />
      </div>

      <div class={clsx(styles.formRow, styles.full)}>
        <TextInput
          name="recommendedHours"
          label="Recommended Replace Hours"
          type="text"
          value={formState.recommendedReplaceHours?.toString() || ""}
          onInput={(value) => {
            const num = parseInt(value, 10);
            setFormState("recommendedReplaceHours", Number.isNaN(num) ? undefined : num);
          }}
        />
      </div>

      <div class={styles.formActions}>
        <Button
          type="button"
          variant="tertiary"
          onClick={handleClose}
          disabled={createCustomStylusMutation.isPending}
        >
          Cancel
        </Button>
        <Button type="submit" variant="primary" disabled={createCustomStylusMutation.isPending}>
          {createCustomStylusMutation.isPending ? "Creating..." : "Create & Add"}
        </Button>
      </div>
    </form>
  );
};

export default CreateStylusModal;
