import React from "react";
import AppLogo from "./app-logo";

export default {
  component: AppLogo,
  title: "AppLogo",
  excludeStories: /.*Data$/,
};

export const Default = () => <AppLogo />;
