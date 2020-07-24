# frozen_string_literal: true

module Types
  module InputObjects
    class NoteOrder < Types::InputObjects::Base
      argument :field, Types::Enums::NoteOrderField, required: true
      argument :direction, Types::Enums::OrderDirection, required: true
    end
  end
end
