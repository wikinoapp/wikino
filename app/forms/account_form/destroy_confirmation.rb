# typed: true
# frozen_string_literal: true

module AccountForm
  class DestroyConfirmation < ApplicationForm
    sig { returns(T.nilable(UserRecord)) }
    attr_accessor :user_record

    attribute :user_atname, :string

    validates :user_record, presence: true
    validates :user_atname, presence: true
    validate :user_atname_correct

    sig { void }
    private def user_atname_correct
      return if user_record.nil?
      return if user_record.not_nil!.atname == user_atname

      errors.add(:user_atname, :incorrect)
    end
  end
end
