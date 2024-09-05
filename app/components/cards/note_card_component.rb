# typed: strict
# frozen_string_literal: true

module Cards
  class NoteCardComponent < ApplicationComponent
    sig { params(note: Note, class_name: String).void }
    def initialize(note:, class_name: "")
      @note = note
      @class_name = class_name
    end

    sig { returns(Note) }
    attr_reader :note
    private :note

    sig { returns(String) }
    attr_reader :class_name
    private :class_name
  end
end
