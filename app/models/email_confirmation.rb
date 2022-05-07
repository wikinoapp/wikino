# frozen_string_literal: true

class EmailConfirmation < ApplicationRecord
  enum event: { sign_in: 0, sign_up: 1 }, _prefix: true

  encrypts :email, deterministic: true, ignore_case: true

  attribute :token, :string, default: -> { SecureRandom.uuid }
  attribute :expires_at, :datetime, default: -> { Time.zone.now + 2.hours }
  attribute :state, :string

  validates :email, presence: true, email: true
  validates :event, presence: true
  validates :token, presence: true, format: { with: /\A[a-zA-Z0-9-]+\z/ }
  validates :expires_at, presence: true
  validates :state, presence: true, format: { with: /\A[a-zA-Z0-9]+\z/ }

  def user_exists?
    User.only_kept.where(email:).exists?
  end

  def url(state:)
    @url ||= [ENV.fetch('NONOTO_WEB_URL'), "/email_confirmation/callback?", { token:, state: }.to_query].join
  end

  def subject
    @subject ||= I18n.t("email_confirmation_mailer.#{event}_event.subject")
  end

  def graphql
    @graphql ||= Graphql.new(self)
  end

  class Graphql
    def initialize(email_confirmation)
      @email_confirmation = email_confirmation
    end

    def user_not_found_error
      {
        email_confirmation: nil,
        errors: [{
          message: "User not found. Sign up instead?",
          code: "USER_NOT_FOUND"
        }]
      }
    end

    def invalid_error
      {
        email_confirmation: nil,
        errors: @email_confirmation.errors.full_messages.map { |message| { message: } }
      }
    end

    def success
      {
        email_confirmation: @email_confirmation,
        errors: []
      }
    end
  end
end
