# typed: strict
# frozen_string_literal: true

module BaseUI
  class BreadcrumbComponent < ApplicationComponent
    renders_many :items, BaseUI::BreadcrumbComponent::Item
  end
end
