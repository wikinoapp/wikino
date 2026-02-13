# typed: strict
# frozen_string_literal: true

module Profiles
  class EditForm < ApplicationForm
    include ActiveModel::Validations::Callbacks

    include FormConcerns::UserAtnameValidatable

    sig { returns(T.nilable(UserRecord)) }
    attr_accessor :user_record

    attribute :atname, :string
    attribute :name, :string, default: ""
    attribute :description, :string, default: ""

    before_validation :convert_nil_to_empty_string

    validates :name, length: {maximum: User::NAME_MAX_LENGTH}
    validates :description, length: {maximum: User::DESCRIPTION_MAX_LENGTH}
    validate :atname_uniqueness

    sig { void }
    private def convert_nil_to_empty_string
      self.name = "" if name.nil?
      self.description = "" if description.nil?
    end

    sig { void }
    private def atname_uniqueness
      return if user_record.nil?
      return if atname.nil?

      if UserRecord.where.not(id: user_record.not_nil!.id).exists?(atname:)
        errors.add(:atname, :uniqueness)
      end
    end
  end
end
