# typed: strict
# frozen_string_literal: true

module Sidebars
  class NoteSidebarComponent < ApplicationComponent
    T::Sig::WithoutRuntime.sig do  params(
      linked_notes: Note::PrivateRelation,
      backlinked_notes: Note::PrivateRelation
    ).void
    end
    def initialize(linked_notes:, backlinked_notes:)
      @linked_notes = linked_notes
      @backlinked_notes = backlinked_notes
    end

    T::Sig::WithoutRuntime.sig { returns(Note::PrivateRelation) }
    attr_reader :linked_notes
    private :linked_notes

    T::Sig::WithoutRuntime.sig { returns(Note::PrivateRelation) }
    attr_reader :backlinked_notes
    private :backlinked_notes
  end
end
