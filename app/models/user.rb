# frozen_string_literal: true

# == Schema Information
#
# Table name: users
#
#  id         :uuid             not null, primary key
#  deleted_at :datetime
#  email      :citext           not null
#  created_at :datetime         not null
#  updated_at :datetime         not null
#
# Indexes
#
#  index_users_on_deleted_at  (deleted_at)
#  index_users_on_email       (email) UNIQUE
#

class User < ApplicationRecord
  include SoftDeletable

  has_many :oauth_providers, dependent: :destroy

  def self.find_by_session(session)
    find_by(id: session[:user_id])
  end

  def self.sign_up_with_google!(oauth_auth:, oauth_params:)
    email = oauth_auth.dig("info", "email")
    user = User.new(email: email)

    transaction do
      user.save!
      user.oauth_providers.create!(
        name: :google,
        uid: oauth_auth["uid"],
        token: oauth_auth.dig("credentials", "token"),
        token_expires_at: oauth_auth.dig("credentials", "expires_at")
      )
    end

    user
  end
end
