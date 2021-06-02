import React, {PropsWithChildren} from "react";
import {createStyles, makeStyles, Theme} from "@material-ui/core/styles";
import classNames from 'classnames';

const useStyles = makeStyles((theme: Theme) =>
  createStyles({
    root: {
      boxShadow: "0 1px 4px rgba(0, 0, 0, 0.18)",
      borderRadius: 8,
      position: "relative",
      backgroundColor: theme.palette.background.paper,
    },
    sharp: {
      borderRadius: 0,
    }
  })
);

function AppCard(
  {
    children, className, sharp, ...props
  }: PropsWithChildren<{
    className?: string,
    sharp?: boolean,
  }>
) {
  const classes = useStyles();
  return (
    <div
      className={classNames(classes.root, className, {[classes.sharp]: sharp})}
      {...props}
    >
      {children}
    </div>
  );
}

export default AppCard;
