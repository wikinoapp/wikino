# typed: strict
# frozen_string_literal: true

module Emails
  class EditForm < ApplicationForm
    attribute :new_email, :string

    validates :new_email, email: true, presence: true
    validate :email_uniqueness

    sig { void }
    private def email_uniqueness
      if UserRecord.find_by(email: new_email)
        errors.add(:new_email, :uniqueness)
      end
    end
  end
end