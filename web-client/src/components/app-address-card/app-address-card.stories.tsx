import React from 'react';
import AppAddressCard, { AppAddressCardProps} from './app-address-card';
import {Story} from '@storybook/react';

export default {
  title: 'AppAddressCard',
  component: AppAddressCard,
}

const Template: Story<AppAddressCardProps> = (args) => (<AppAddressCard {...args}/>);

export const Inactive = Template.bind({});
Inactive.args = {
  name: 'Poseidon',
  address: '03b805fab5e8ec2eee92496925b8067a5d5da4234c0dfc7a10e86ac08e4fa8ccea',
  coins: 1000,
  votes: 15,
  active: false,
};

export const Active = Template.bind({});
Active.args = {
  ...Inactive.args,
  active: true,
};
