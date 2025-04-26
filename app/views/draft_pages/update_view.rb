# typed: strict
# frozen_string_literal: true

module DraftPages
  class UpdateView < ApplicationView
    sig do
      params(
        draft_page: DraftPage,
        link_list: LinkList,
        backlink_list: BacklinkList
      ).void
    end
    def initialize(draft_page:, link_list:, backlink_list:)
      @draft_page = draft_page
      @link_list = link_list
      @backlink_list = backlink_list
    end

    sig { returns(DraftPage) }
    attr_reader :draft_page
    private :draft_page

    sig { returns(LinkList) }
    attr_reader :link_list
    private :link_list

    sig { returns(BacklinkList) }
    attr_reader :backlink_list
    private :backlink_list

    delegate :page, to: :draft_page
  end
end
