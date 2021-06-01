import React from "react";
import AppLogoGroup from "./app-logo-group";

export default {
  component: AppLogoGroup,
  title: "AppLogoGroup",
  excludeStories: /.*Data$/,
};

export const Small = () => <AppLogoGroup />;
export const Large = () => <AppLogoGroup large />;
