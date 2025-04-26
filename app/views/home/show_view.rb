# typed: strict
# frozen_string_literal: true

module Home
  class ShowView < ApplicationView
    sig { params(active_spaces: SpaceRecord::PrivateCollectionProxy, current_user_entity: UserEntity).void }
    def initialize(active_spaces:, current_user_entity:)
      @active_spaces = active_spaces
      @current_user_entity = current_user_entity
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.home.show")
      helpers.set_meta_tags(title:, **default_meta_tags)
    end

    sig { returns(SpaceRecord::PrivateCollectionProxy) }
    attr_reader :active_spaces
    private :active_spaces

    sig { returns(UserEntity) }
    attr_reader :current_user_entity
    private :current_user_entity

    sig { returns(PageName) }
    private def current_page_name
      PageName::Home
    end
  end
end
