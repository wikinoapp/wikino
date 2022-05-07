# frozen_string_literal: true

class User < ApplicationRecord
  include SoftDeletable

  has_many :notes, dependent: :destroy
  has_one :access_token, dependent: :destroy

  validates :email,
    presence: true,
    uniqueness: { case_sensitive: false },
    email: true
end
