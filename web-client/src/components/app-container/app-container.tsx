import React from "react";
import { withStyles } from "@material-ui/core";

const styles = (theme) => ({
  container: {
    padding: theme.spacing(3),
    display: "flex",
    alignItems: "center",
    minHeight: "100%",
  },
  content: {
    width: "100%",
  },
});

function AppContainerRaw({ classes, children }) {
  return (
    <div className={classes.container}>
      <div className={classes.content}>{children}</div>
    </div>
  );
}

const AppContainer = withStyles(styles)(AppContainerRaw);

export default AppContainer;
