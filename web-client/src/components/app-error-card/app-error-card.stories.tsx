import React from "react";
import AppErrorCard from "./app-error-card";

export default {
  component: AppErrorCard,
  title: "AppErrorCard",
};

export const Default = () => (
  <AppErrorCard>The password is too long</AppErrorCard>
);
export const Overflow = () => (
  <AppErrorCard>
    The error card content is tooooooo long for such small screen, especially
    this small, or that big (like my ig)
  </AppErrorCard>
);
