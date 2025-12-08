# typed: strict
# frozen_string_literal: true

module Breadcrumbs
  class SettingsComponent < ApplicationComponent
    renders_many :items, BaseUI::BreadcrumbComponent::Item
  end
end
