import React from "react";
import withStyles from "@material-ui/core/styles/withStyles";

const styles = (theme) => ({
  main: {
    marginBottom: theme.spacing(0.5),
  },
  label: {
    color: theme.palette.text.secondary,
    fontSize: "0.6rem",
  },
  content: {
    color: theme.palette.text.primary,
    fontSize: "0.85rem",
    wordWrap: "break-word",
  },
});

function AppInfoLineRaw({ classes, label, children }) {
  return (
    <div className={classes.main}>
      <div className={classes.label}>{label}</div>
      <div className={classes.content}>{children}</div>
    </div>
  );
}

//@ts-ignore
const AppInfoLine = withStyles(styles)(AppInfoLineRaw);

export default AppInfoLine;
