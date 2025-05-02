# typed: strict
# frozen_string_literal: true

module AccountForm
  class Creation < ApplicationForm
    include FormConcerns::UserAtnameValidatable
    include FormConcerns::PasswordValidatable

    attribute :atname, :string
    attribute :email, :string
    attribute :locale, :string
    attribute :password, :string
    attribute :time_zone, :string

    validates :email, email: true, presence: true
    validates :locale, inclusion: {in: Locale.values.map(&:serialize)}, presence: true
    validates :time_zone,
      format: {with: %r{\A[A-Za-z]+/[A-Za-z_]+\z}},
      presence: true
    validate :atname_uniqueness
    validate :email_uniqueness

    sig { void }
    private def atname_uniqueness
      return if atname.nil?

      if UserRecord.exists?(atname:)
        errors.add(:atname, :uniqueness)
      end
    end

    sig { void }
    private def email_uniqueness
      return if email.nil?

      if UserRecord.exists?(email:)
        errors.add(:email, :uniqueness)
      end
    end
  end
end
