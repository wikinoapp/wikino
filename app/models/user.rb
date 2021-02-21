# frozen_string_literal: true
# == Schema Information
#
# Table name: users
#
#  id           :uuid             not null, primary key
#  access_token :string           not null
#  deleted_at   :datetime
#  email        :citext           not null
#  signed_up_at :datetime         not null
#  created_at   :datetime         not null
#  updated_at   :datetime         not null
#
# Indexes
#
#  index_users_on_access_token  (access_token) UNIQUE
#  index_users_on_email         (email) UNIQUE
#
class User < ApplicationRecord
  include SoftDeletable

  # Include default devise modules. Others available are:
  # :lockable, :timeoutable, :trackable and :omniauthable
  # devise :confirmable, :database_authenticatable, :recoverable, :registerable, :rememberable, :validatable
  has_secure_token :access_token

  has_many :notes, dependent: :destroy

  validates :email,
    presence: true,
    uniqueness: { case_sensitive: false },
    email: true
end
