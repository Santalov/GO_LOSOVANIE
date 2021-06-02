import React from "react";
import AppInfoLine from "./app-info-line";

export default {
  component: AppInfoLine,
  title: "AppInfoLine",
};

export const Default = () => (
  <AppInfoLine label="name">This is contract name</AppInfoLine>
);

export const Large = () => (
  <AppInfoLine label="name" large>This is contract name</AppInfoLine>
);
