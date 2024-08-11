# typed: strict
# frozen_string_literal: true

class User < ApplicationRecord
  extend T::Sig

  include Discard::Model

  ATNAME_FORMAT = /\A[A-Za-z0-9_]+\z/
  ATNAME_MIN_LENGTH = 2
  ATNAME_MAX_LENGTH = 20

  enum :role, {
    Role::Owner.serialize => 0
  }, prefix: true

  enum :locale, {
    Locale::En.serialize => 0,
    Locale::Ja.serialize => 1
  }, prefix: true

  belongs_to :space
  has_many :sessions, dependent: :restrict_with_exception
  has_one :user_password, dependent: :restrict_with_exception

  delegate :identifier, to: :space, prefix: :space

  sig do
    params(
      email: String,
      atname: String,
      password: String,
      locale: Locale,
      time_zone: String,
      current_time: ActiveSupport::TimeWithZone
    ).returns(User)
  end
  def self.create_initial_user!(email:, atname:, password:, locale:, time_zone:, current_time:)
    user = create!(
      email:,
      atname:,
      role: Role::Owner.serialize,
      locale: locale.serialize,
      time_zone:,
      joined_at: current_time
    )
    user.create_user_password!(space: user.space, password:)

    user
  end

  sig { returns(Locale) }
  def deserialized_locale
    Locale.deserialize(locale)
  end

  T::Sig::WithoutRuntime.sig { params(note: Note).returns(Note::PrivateRelation) }
  def notes_except(note)
    note.new_record? ? notes : notes.where.not(id: note.id)
  end

  sig { params(email_confirmation: EmailConfirmation).void }
  def run_after_email_confirmation_success!(email_confirmation:)
    return unless email_confirmation.succeeded?

    if email_confirmation.deserialized_event == EmailConfirmationEvent::EmailUpdate
      update!(email: email_confirmation.email)
    end

    nil
  end
end
