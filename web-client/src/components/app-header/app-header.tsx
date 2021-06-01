import React from "react";
import withStyles from "@material-ui/core/styles/withStyles";
import AppLogoGroup from "../app-logo-group/app-logo-group";

const styles = (theme) => ({
  header: {
    display: "grid",
    gridTemplateColumns: "1fr auto",
    gridColumnGap: theme.spacing(1),
    fontSize: "1.8rem",
    marginBottom: theme.spacing(3),
  },
});

function AppHeaderRaw(props: {
  classes: any;
  className?: string;
  children: any;
}) {
  const { classes, className, children } = props;
  return (
    <header className={classes.header + (className ? " " + className : "")}>
      <div>{children}</div>
      <AppLogoGroup />
    </header>
  );
}

const AppHeader = withStyles(styles)(AppHeaderRaw);

export default AppHeader;
