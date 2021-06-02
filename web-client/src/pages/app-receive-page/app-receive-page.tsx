import AppHeadline from '../../components/app-headline/app-headline';
import AppInfoLine from '../../components/app-info-line/app-info-line';
import {createStyles, makeStyles, TextField, Theme, Typography} from '@material-ui/core';
import {forms} from '../../theme/app-theme-constants';
import AppButton from '../../components/app-button/app-button';
import React, {SyntheticEvent} from 'react';
import AppButtonContainer from '../../components/app-button-container/app-button-container';
import {useFormik} from 'formik';
import * as yup from 'yup';

const useStyles = makeStyles((theme: Theme) =>
  createStyles({
    field: {
      marginBottom: theme.spacing(forms.field)
    },
    lastField: {
      marginBottom: theme.spacing(forms.lastField)
    }
  })
);

const validationSchema = yup.object({
  amount: yup
    .number()
    .required('Введите сумму'),
});

function AppReceivePage() {
  const classes = useStyles();
  const copyAddress = (e: SyntheticEvent) => {
    e.preventDefault();
    navigator
      .clipboard
      .writeText('address')
      .then(() => {
          console.log('copied', 'address')
        }
      );
  };
  const formik = useFormik({
    initialValues: {
      amount: '',
    },
    validationSchema: validationSchema,
    onSubmit: (values) => {
      alert(JSON.stringify(values));
    }
  });
  return (
    <>
      <AppHeadline>Получить</AppHeadline>
      <div
        className={classes.field}
      >
        <AppInfoLine
          label="Ваш адрес для получения монет и участия в голосованиях"
          large
        >
          03b805fab5e8ec2eee92496925b8067a
        </AppInfoLine>
      </div>
      <AppButton
        onClick={copyAddress}
        className={classes.lastField}
      >
        Скопировать
      </AppButton>
      <Typography
        variant="h5"
        className={classes.field}
      >
        Запрос монет
      </Typography>
      <form onSubmit={formik.handleSubmit}>
        <TextField
          required
          id="amount"
          label="Сумма"
          placeholder="Целое число"
          type="number"
          fullWidth
          variant="outlined"
          className={classes.lastField}
          value={formik.values.amount}
          onChange={formik.handleChange}
          error={formik.touched.amount && !!formik.errors.amount}
          helperText={formik.touched.amount && formik.errors.amount}
        >
        </TextField>
        <AppButtonContainer>
          <AppButton
            type="submit"
          >
            Запросить
          </AppButton>
        </AppButtonContainer>
      </form>
    </>
  )
}

export default AppReceivePage
