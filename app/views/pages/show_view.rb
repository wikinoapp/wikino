# typed: strict
# frozen_string_literal: true

module Pages
  class ShowView < ApplicationView
    sig do
      params(
        current_user: T.nilable(User),
        page: Page,
        link_list: LinkList,
        backlink_list: BacklinkList
      ).void
    end
    def initialize(current_user:, page:, link_list:, backlink_list:)
      @current_user = current_user
      @page = page
      @link_list = link_list
      @backlink_list = backlink_list
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.pages.show", space_name: space.name, page_title: page.title)
      helpers.set_meta_tags(title:, **default_meta_tags(site: false))
    end

    sig { returns(T.nilable(User)) }
    attr_reader :current_user
    private :current_user

    sig { returns(Page) }
    attr_reader :page
    private :page

    sig { returns(LinkList) }
    attr_reader :link_list
    private :link_list

    sig { returns(BacklinkList) }
    attr_reader :backlink_list
    private :backlink_list

    delegate :space, :topic, to: :page

    sig { returns(T::Boolean) }
    private def signed_in?
      !current_user.nil?
    end

    sig { returns(PageName) }
    private def current_page_name
      PageName::PageDetail
    end
  end
end
