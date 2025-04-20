# typed: true
# frozen_string_literal: true

class UserSession < ApplicationModel
  TOKENS_COOKIE_KEY = :user_session_tokens

  sig { returns(String) }
  attr_accessor :email

  sig { returns(String) }
  attr_accessor :password

  sig { returns(User) }
  attr_accessor :user

  sig { returns(String) }
  attr_accessor :token

  sig { returns(String) }
  attr_accessor :ip_address

  sig { returns(String) }
  attr_accessor :user_agent

  sig { returns(ActiveSupport::TimeWithZone) }
  attr_accessor :signed_in_at

  sig { params(attributes: T.untyped).void }
  def initialize(attributes = {})
    super
    @ip_address ||= "unknown"
    @user_agent ||= "unknown"
  end

  validates :ip_address, presence: true
  validates :user_agent, presence: true

  with_options(on: :create) do
    validates :email, email: true, presence: true
    validates :password, presence: true
  end

  with_options(on: :read) do
    validates :user, presence: true
    validates :token, presence: true
    validates :signed_in_at, presence: true
  end
end
