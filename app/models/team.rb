# frozen_string_literal: true

# == Schema Information
#
# Table name: teams
#
#  id         :bigint           not null, primary key
#  name       :string           default(""), not null
#  created_at :datetime         not null
#  updated_at :datetime         not null
#
class Team < ApplicationRecord
  has_many :projects, dependent: :destroy
  has_many :team_members, dependent: :destroy
  has_many :users, through: :team_members
  has_many :notes, dependent: :destroy
end
