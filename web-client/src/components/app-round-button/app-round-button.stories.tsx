import React from "react";
import AppRoundButton from "./app-round-button"

export default {
  component: AppRoundButton,
  title: "AppRoundButton",
  excludeStories: /.*Data$/,
};

export const Default = () => <AppRoundButton>1</AppRoundButton>;
export const Disabled = () => <AppRoundButton disabled>0</AppRoundButton>;
