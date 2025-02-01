# typed: strict
# frozen_string_literal: true

module Pages
  class EditView < ApplicationView
    sig do
      params(
        space: Space,
        page: Page,
        form: EditPageForm,
        link_collection: LinkCollection,
        backlink_collection: BacklinkCollection,
        draft_page: T.nilable(DraftPage)
      ).void
    end
    def initialize(space:, page:, form:, link_collection:, backlink_collection:, draft_page: nil)
      @space = space
      @page = page
      @form = form
      @link_collection = link_collection
      @backlink_collection = backlink_collection
      @draft_page = draft_page
    end

    sig { returns(Space) }
    attr_reader :space
    private :space

    sig { returns(Page) }
    attr_reader :page
    private :page

    sig { returns(EditPageForm) }
    attr_reader :form
    private :form

    sig { returns(LinkCollection) }
    attr_reader :link_collection
    private :link_collection

    sig { returns(BacklinkCollection) }
    attr_reader :backlink_collection
    private :backlink_collection

    sig { returns(T.nilable(DraftPage)) }
    attr_reader :draft_page
    private :draft_page
  end
end
