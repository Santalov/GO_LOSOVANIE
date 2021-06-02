import React, {PropsWithChildren} from "react";
import ButtonBase from "@material-ui/core/ButtonBase";
import {createStyles, makeStyles, Theme} from "@material-ui/core/styles";
import {ButtonProps} from '@material-ui/core';
import classNames from 'classnames';

const useStyles = makeStyles((theme: Theme) =>
  createStyles({
    root: {
      position: "relative",
      border: "2px solid",
      borderColor: theme.palette.primary.main,
      borderRadius: 6,
      flexShrink: 0,
      height: 40,
      lineHeight: "36px",
      paddingLeft: 30,
      paddingRight: 30,
      paddingTop: 0,
      paddingBottom: 0,
      fontSize: "1rem",
      color: theme.palette.primary.main,
    },
    disabled: {
      borderColor: theme.palette.text.disabled,
      color: theme.palette.text.secondary,
    },
  })
);

function AppButton(
  {
    children,
    className,
    disabled,
    ...props
  }: PropsWithChildren<{
    className?: string,
    disabled?: boolean
  }> & ButtonProps
) {
  const classes = useStyles();
  return (
    <ButtonBase
      {...props}
      className={
        classNames(classes.root, {[classes.disabled]: disabled}, className)
      }
    >
      {children}
    </ButtonBase>
  );
}

export default AppButton;
