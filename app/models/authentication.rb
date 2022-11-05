# typed: true
# frozen_string_literal: true

class Authentication
  extend T::Sig

  include ActiveModel::Model

  attr_accessor :auth0_user_id

  validates :auth0_user_id, presence: true

  def find_or_create_user!
    User.find_or_create_by!(auth0_user_id:)
  end
end
