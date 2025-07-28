# typed: true
# frozen_string_literal: true

module Accounts
  class DestroyConfirmationForm < ApplicationForm
    sig { returns(T.nilable(UserRecord)) }
    attr_accessor :user_record

    attribute :user_atname, :string

    validates :user_record, presence: true
    validates :user_atname, presence: true
    validate :user_atname_correct
    validate :user_has_no_active_spaces

    sig { void }
    private def user_atname_correct
      return if user_record.nil?
      return if user_record.not_nil!.atname == user_atname

      errors.add(:user_atname, :incorrect)
    end

    sig { void }
    private def user_has_no_active_spaces
      return if user_record.nil?
      return unless user_record.not_nil!.active_space_records.exists?

      errors.add(:user_record, :has_active_spaces)
    end
  end
end
