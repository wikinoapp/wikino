# typed: strict
# frozen_string_literal: true

module Profiles
  class ShowView < ApplicationView
    sig do
      params(
        current_user_entity: T.nilable(UserEntity),
        user_entity: UserEntity,
        joined_space_entities: T::Array[SpaceEntity]
      ).void
    end
    def initialize(current_user_entity:, user_entity:, joined_space_entities:)
      @current_user_entity = current_user_entity
      @user_entity = user_entity
      @joined_space_entities = joined_space_entities
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.profiles.show", name: user_entity.name, atname: user_entity.atname)
      helpers.set_meta_tags(title:, **default_meta_tags)
    end

    sig { returns(T.nilable(UserEntity)) }
    attr_reader :current_user_entity
    private :current_user_entity

    sig { returns(UserEntity) }
    attr_reader :user_entity
    private :user_entity

    sig { returns(T::Array[SpaceEntity]) }
    attr_reader :joined_space_entities
    private :joined_space_entities

    sig { returns(T::Boolean) }
    private def signed_in?
      !current_user_entity.nil?
    end

    sig { returns(T::Boolean) }
    private def can_edit_profile?
      signed_in? && current_user_entity.not_nil!.database_id == user_entity.database_id
    end

    sig { returns(PageName) }
    private def current_page_name
      PageName::Profile
    end
  end
end
