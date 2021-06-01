import React from "react";
import makeStyles from "@material-ui/core/styles/makeStyles";
import { withTheme } from "@material-ui/core";

function AppBackgroundRaw({ theme, children }) {
  const appClass = makeStyles({
    root: {
      backgroundColor: theme.palette.background.default,
      color: theme.palette.text.primary,
      height: "100%",
      overflow: "auto",
    },
  })().root;
  return <div className={appClass}>{children}</div>;
}

const AppBackground = withTheme(AppBackgroundRaw);

export default AppBackground;
