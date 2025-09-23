# typed: strict
# frozen_string_literal: true

module EditSuggestionPages
  class CreateForm < ApplicationForm
    sig { returns(T.nilable(SpaceMemberRecord)) }
    attr_accessor :space_member_record

    sig { returns(T.nilable(PageRecord)) }
    attr_accessor :page_record

    sig { returns(T.nilable(EditSuggestionRecord)) }
    attr_accessor :edit_suggestion_record

    attribute :page_title, :string
    attribute :page_body, :string, default: ""

    validates :space_member_record, presence: true
    validates :edit_suggestion_record, presence: true
    validates :page_title, presence: true, length: {maximum: 255}
  end
end
