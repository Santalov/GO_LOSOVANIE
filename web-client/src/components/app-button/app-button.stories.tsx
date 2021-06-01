import React from "react";
import AppButton from "./app-button";

export default {
  component: AppButton,
  title: "Button",
  excludeStories: /.*Data$/,
};

export const Default = () => <AppButton>Button</AppButton>;
export const LongContent = () => (
  <AppButton>Super Big Content Inside a Button</AppButton>
);
export const Disabled = () => <AppButton disabled>Disabled</AppButton>;
