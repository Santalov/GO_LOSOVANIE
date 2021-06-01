import React from "react";
import AppCard from "./app-card";
import { makeStyles } from "@material-ui/core/styles";

export default {
  component: AppCard,
  title: "AppCard",
  excludeStories: /.*Data$/,
};

const useStyles = makeStyles({
  root: {
    padding: "20px",
  },
});

export const Default = () => {
  const className = useStyles().root;
  return (
    <AppCard className={className}>
      <h1>Headline</h1>
      <p>Some text</p>
    </AppCard>
  );
};
