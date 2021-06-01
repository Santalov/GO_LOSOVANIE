import React from "react";
import { makeStyles } from "@material-ui/core/styles";
import withTheme from "@material-ui/core/styles/withTheme";

function AppCardRaw({ children, className, theme, ...props }) {
  const useStyles = makeStyles({
    root: {
      boxShadow: "0 1px 4px rgba(0, 0, 0, 0.18)",
      borderRadius: 8,
      position: "relative",
      backgroundColor: theme.palette.background.paper,
    },
  });
  const classes = useStyles();
  return (
    <div className={classes.root + " " + className} {...props}>
      {children}
    </div>
  );
}

const AppCard = withTheme(AppCardRaw);

export default AppCard;
