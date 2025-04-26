# typed: strict
# frozen_string_literal: true

module Profiles
  class ShowView < ApplicationView
    sig do
      params(
        current_user: T.nilable(User),
        user: User,
        joined_spaces: T::Array[Space]
      ).void
    end
    def initialize(current_user:, user:, joined_spaces:)
      @current_user = current_user
      @user = user
      @joined_spaces = joined_spaces
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.profiles.show", name: user.name, atname: user.atname)
      helpers.set_meta_tags(title:, **default_meta_tags)
    end

    sig { returns(T.nilable(User)) }
    attr_reader :current_user
    private :current_user

    sig { returns(User) }
    attr_reader :user
    private :user

    sig { returns(T::Array[Space]) }
    attr_reader :joined_spaces
    private :joined_spaces

    sig { returns(T::Boolean) }
    private def signed_in?
      !current_user.nil?
    end

    sig { returns(T::Boolean) }
    private def can_edit_profile?
      signed_in? && current_user.not_nil!.database_id == user.database_id
    end

    sig { returns(PageName) }
    private def current_page_name
      PageName::Profile
    end
  end
end
