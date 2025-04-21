# typed: strict
# frozen_string_literal: true

module Home
  class ShowView < ApplicationView
    sig { params(active_space_entities: T::Array[SpaceEntity], current_user_entity: UserEntity).void }
    def initialize(active_space_entities:, current_user_entity:)
      @active_space_entities = active_space_entities
      @current_user_entity = current_user_entity
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.home.show")
      helpers.set_meta_tags(title:, **default_meta_tags)
    end

    sig { returns(T::Array[Space]) }
    attr_reader :active_space_entities
    private :active_space_entities

    sig { returns(UserEntity) }
    attr_reader :current_user_entity
    private :current_user_entity

    sig { returns(PageName) }
    private def current_page_name
      PageName::Home
    end
  end
end
