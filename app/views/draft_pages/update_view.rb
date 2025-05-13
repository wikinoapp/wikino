# typed: strict
# frozen_string_literal: true

module DraftPages
  class UpdateView < ApplicationView
    sig do
      params(
        current_user: User,
        draft_page: DraftPage,
        link_list: LinkList,
        backlink_list: BacklinkList
      ).void
    end
    def initialize(current_user:, draft_page:, link_list:, backlink_list:)
      @current_user = current_user
      @draft_page = draft_page
      @link_list = link_list
      @backlink_list = backlink_list
    end

    sig { returns(User) }
    attr_reader :current_user
    private :current_user

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

    sig { returns(String) }
    private def draft_saved_time
      l(draft_page.modified_at.in_time_zone(current_user.time_zone), format: :hm)
    end
  end
end
