# typed: true
# frozen_string_literal: true

class SpaceDestroyConfirmationForm < ApplicationForm
  sig { returns(T.nilable(SpaceRecord)) }
  attr_accessor :space_record

  attribute :space_name, :string

  validates :space_record, presence: true
  validates :space_name, presence: true
  validate :space_name_correct

  sig { void }
  private def space_name_correct
    return if space_record.nil?
    return if space_record.not_nil!.name == space_name

    errors.add(:space_name, :incorrect)
  end
end
