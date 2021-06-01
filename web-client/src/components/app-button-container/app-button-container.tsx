import React from "react";
import withStyles from "@material-ui/core/styles/withStyles";

const styles = (theme) => ({
  container: {
    display: "flex",
    justifyContent: "flex-end",
    alignItems: "stretch",
  },
});

function AppButtonContainerRaw({ classes, children, ...props }) {
  return (
    <div className={classes.container} {...props}>
      {children}
    </div>
  );
}

const AppButtonContainer = withStyles(styles)(AppButtonContainerRaw);

export default AppButtonContainer;
