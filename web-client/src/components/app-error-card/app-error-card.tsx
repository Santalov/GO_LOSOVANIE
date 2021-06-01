import React from "react";
import withStyles from "@material-ui/core/styles/withStyles";

const styles = (theme) => ({
  errorCard: {
    padding: theme.spacing(1),
    // backgroundColor: theme.palette.error.main,
    border: "1px solid " + theme.palette.error.dark,
    display: "flex",
    alignItems: "center",
    alignContent: "center",
    fontSize: "0.9rem",
    borderRadius: 4,
  },
});

function AppErrorCardRaw({ classes, className = '', children }) {
  return (
    <div className={classes.errorCard + (className ? " " + className : "")}>
      {children}
    </div>
  );
}

const AppErrorCard = withStyles(styles)(AppErrorCardRaw);

export default AppErrorCard;
