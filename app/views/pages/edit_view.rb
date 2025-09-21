# typed: strict
# frozen_string_literal: true

module Pages
  class EditView < ApplicationView
    sig do
      params(
        current_user: User,
        space: Space,
        page: Page,
        form: Pages::EditForm,
        link_list: LinkList,
        backlink_list: BacklinkList,
        topic: Topic,
        can_create_edit_suggestion: T::Boolean,
        existing_edit_suggestions: T::Array[EditSuggestion],
        draft_page: T.nilable(DraftPage)
      ).void
    end
    def initialize(
      current_user:,
      space:,
      page:,
      form:,
      link_list:,
      backlink_list:,
      topic:,
      can_create_edit_suggestion:,
      existing_edit_suggestions:,
      draft_page: nil
    )
      @space = space
      @page = page
      @form = form
      @link_list = link_list
      @backlink_list = backlink_list
      @current_user = current_user
      @draft_page = draft_page
      @topic = topic
      @can_create_edit_suggestion = can_create_edit_suggestion
      @existing_edit_suggestions = existing_edit_suggestions
    end

    sig { override.void }
    def before_render
      helpers.set_meta_tags(title: "#{title} | #{space.name}", **default_meta_tags)
    end

    sig { returns(User) }
    attr_reader :current_user
    private :current_user

    sig { returns(Space) }
    attr_reader :space
    private :space

    sig { returns(Page) }
    attr_reader :page
    private :page

    sig { returns(Pages::EditForm) }
    attr_reader :form
    private :form

    sig { returns(LinkList) }
    attr_reader :link_list
    private :link_list

    sig { returns(BacklinkList) }
    attr_reader :backlink_list
    private :backlink_list

    sig { returns(T.nilable(DraftPage)) }
    attr_reader :draft_page
    private :draft_page

    sig { returns(Topic) }
    attr_reader :topic
    private :topic

    sig { returns(T::Boolean) }
    attr_reader :can_create_edit_suggestion
    private :can_create_edit_suggestion

    sig { returns(T::Array[EditSuggestion]) }
    attr_reader :existing_edit_suggestions
    private :existing_edit_suggestions

    sig { returns(String) }
    private def title
      I18n.t("meta.title.pages.edit")
    end

    sig { returns(PageName) }
    private def current_page_name
      PageName::PageEdit
    end
  end
end
