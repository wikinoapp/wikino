# typed: strict
# frozen_string_literal: true

class EmailConfirmationRecord < ApplicationRecord
  CODE_LENGTH = T.let(6, Integer)
  EXPIRES_IN = T.let(15.minutes, ActiveSupport::Duration)

  self.table_name = "email_confirmations"

  enum :event, {
    EmailConfirmationEvent::SignUp.serialize => 0,
    EmailConfirmationEvent::EmailUpdate.serialize => 1,
    EmailConfirmationEvent::PasswordReset.serialize => 2
  }, prefix: true

  scope :active, -> { where(succeeded_at: nil).where("started_at > ?", EXPIRES_IN.ago) }
  scope :succeeded, -> { where.not(succeeded_at: nil) }

  validates :email, presence: true
  validates :code, format: {with: /\A[A-Z0-9]{#{CODE_LENGTH}}\z/o}, presence: true

  sig { returns(String) }
  def self.generate_code
    characters = ("0".."9").to_a + ("A".."Z").to_a
    Array.new(CODE_LENGTH) { characters.sample }.join
  end

  sig { returns(EmailConfirmationEvent) }
  def deserialized_event
    EmailConfirmationEvent.deserialize(event)
  end

  sig { returns(T::Boolean) }
  def succeeded?
    !succeeded_at.nil?
  end

  sig { void }
  def success!
    update!(succeeded_at: Time.current) unless succeeded?

    nil
  end

  sig { params(locale: Locale).void }
  def send_mail!(locale:)
    EmailConfirmationMailer.email_confirmation(email_confirmation_id: id.not_nil!, locale: locale.serialize).deliver_later

    nil
  end
end
