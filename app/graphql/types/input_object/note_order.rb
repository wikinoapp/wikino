# frozen_string_literal: true

module Types
  module InputObject
    class NoteOrder < Types::InputObject::Base
      argument :field, Types::Enum::NoteOrderField, required: true
      argument :direction, Types::Enum::OrderDirection, required: true
    end
  end
end
